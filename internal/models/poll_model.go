package models

import (
	"api_voty/ent"
	"api_voty/ent/poll"
	"api_voty/ent/polloption"
	"api_voty/ent/user"
	"api_voty/ent/vote"
	"context"
	"errors"
	"strconv"
	"time"
)

type PollModel struct {
	client *ent.Client
}

func NewPollModel(client *ent.Client) *PollModel {
	return &PollModel{client: client}
}

func (m *PollModel) CastVote(ctx context.Context, pollIDStr, optionIDStr, userID string) (int, error) { // Iniciamos Transacción (Atomicidad)
	pollID, _ := strconv.Atoi(pollIDStr)
	optionID, _ := strconv.Atoi(optionIDStr)

	tx, err := m.client.Tx(ctx)
	if err != nil {
		return 0, err
	}

	// 1. Verificar si la encuesta está abierta
	p, err := tx.Poll.Query().Where(poll.ID(pollID)).Only(ctx)
	if err != nil || !p.IsOpen {
		tx.Rollback()
		return 0, errors.New("POLL_CLOSED")
	}

	// 2. Verificar si el usuario ya votó (SSOT)
	// El índice único que pusimos en el esquema también protegerá esto
	exists, _ := tx.Vote.Query().
		Where(vote.HasUserWith(user.ID(userID)), vote.HasPollWith(poll.ID(pollID))).
		Exist(ctx)

	if exists {
		tx.Rollback()
		return 0, errors.New("ALREADY_VOTED") // El móvil dispara el Rollback con esto
	}

	// 3. Crear el registro del voto
	_, err = tx.Vote.Create().
		SetUserID(userID).
		SetPollID(pollID).
		SetPollOptionID(optionID).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// 4. Incrementar contador en la opción
	opt, err := tx.PollOption.UpdateOneID(optionID).
		AddVotesCount(1).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Confirmar todo
	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return opt.VotesCount, nil
}


func (m *PollModel) GetByIDWithUserStatus(ctx context.Context, pollID int, userID string) (*ent.Poll, error) {
	return m.client.Poll.
		Query().
		Where(poll.ID(pollID)).
		WithOptions().
		WithVotes(func(q *ent.VoteQuery) {
			q.Where(vote.HasUserWith(user.IDEQ(userID))).WithPollOption()
		}).
		Only(ctx)
}

// Create crea la cabecera de la encuesta
func (m *PollModel) Create(ctx context.Context, title string) (*ent.Poll, error) {
	return m.client.Poll.
		Create().
		SetTitle(title).
		SetIsOpen(true). // La creamos abierta por defecto
		SetCreatedAt(time.Now()).
		Save(ctx)
}

// AddOption añade una opción individual a una encuesta existente
func (m *PollModel) AddOption(ctx context.Context, pollID string, text string) error {
	id, err := strconv.Atoi(pollID)
	if err != nil {
		return err
	}
	return m.client.PollOption.
		Create().
		SetText(text).
		SetPollID(id).
		SetVotesCount(0).
		Exec(ctx)
}

func (m *PollModel) ListAll(ctx context.Context) ([]*ent.Poll, error) {
	return m.client.Poll.
		Query().
		WithOptions(). // Carga las opciones de cada encuesta (Eager Loading)
		Order(ent.Desc(poll.FieldCreatedAt)).
		All(ctx)
}

func (m *PollModel) ListAllWithUserStatus(ctx context.Context, userID string) ([]*ent.Poll, error) {
	return m.client.Poll.
		Query().
		WithOptions().
		WithVotes(func(q *ent.VoteQuery) {
			q.Where(vote.HasUserWith(user.ID(userID))).
				WithPollOption()
		}).
		Order(ent.Desc(poll.FieldCreatedAt)).
		All(ctx)
}

// Update actualiza el título o el estado de una encuesta
func (m *PollModel) Update(ctx context.Context, id int, title string, isOpen bool, options []string) (*ent.Poll, error) {
    // Usamos una transacción porque vamos a tocar varias tablas
    tx, err := m.client.Tx(ctx)
    if err != nil {
        return nil, err
    }

    // 1. Actualizar datos básicos de la encuesta
    p, err := tx.Poll.UpdateOneID(id).
        SetTitle(title).
        SetIsOpen(isOpen).
        Save(ctx)
    if err != nil {
        tx.Rollback()
        return nil, err
    }
	println(p)

    // 2. Si vienen opciones, sincronizamos (Borrar antiguas y crear nuevas)
    // Nota: Solo haz esto si no hay votos o si decides resetear la encuesta
    if options != nil {
        // Borrar opciones actuales
        _, err = tx.PollOption.Delete().
            Where(polloption.HasPollWith(poll.ID(id))).
            Exec(ctx)
        if err != nil {
            tx.Rollback()
            return nil, err
        }

        // Crear las nuevas opciones
        bulk := make([]*ent.PollOptionCreate, len(options))
        for i, txt := range options {
            bulk[i] = tx.PollOption.Create().SetText(txt).SetPollID(id)
        }
        err = tx.PollOption.CreateBulk(bulk...).Exec(ctx)
        if err != nil {
            tx.Rollback()
            return nil, err
        }
    }

    err = tx.Commit()
    if err != nil {
        return nil, err
    }

    // Devolvemos la encuesta con los cambios cargados (Eager load)
    return m.client.Poll.Query().Where(poll.ID(id)).WithOptions().Only(ctx)
}
// Delete elimina una encuesta y, dependiendo de tu esquema,
// Ent puede manejar el "Cascade Delete" de opciones y votos.
func (m *PollModel) Delete(ctx context.Context, id string) error {
	pollID, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	return m.client.Poll.
		DeleteOneID(pollID).
		Exec(ctx)
}
