package api

import (
	"api_voty/internal/utils"
	"context"

	"github.com/danielgtaylor/huma/v2"
)

type VoteInput struct {
	PollID   string `path:"poll_id" doc:"ID de la encuesta"`
	OptionID string `path:"option_id" doc:"ID de la opción elegida"`
}

func (a *UserAPI) PostVote(ctx context.Context, input *VoteInput) (*struct{}, error) {
	// Obtenemos el ID del usuario desde el JWT (Context)
	userID := utils.GetUserIDFromContext(ctx)

	newCount, err := a.pollModel.CastVote(ctx, input.PollID, input.OptionID, userID)
	if err != nil {
		// Retornamos 403 para que el móvil sepa que debe revertir su estado local
		return nil, huma.Error403Forbidden("Voto rechazado", err)
	}

	// Si todo salió bien, enviamos el broadcast por el Hub
	a.Hub.Broadcast <- VoteUpdate{
		PollID:   input.PollID,
		OptionID: input.OptionID,
		NewCount: newCount,
	}

	return nil, nil
}
