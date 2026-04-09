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
    userID := utils.GetUserIDFromContext(ctx)

    newCount, err := a.pollModel.CastVote(ctx, input.PollID, input.OptionID, userID)
    if err != nil {
        return nil, huma.Error403Forbidden("Voto rechazado", err)
    }

    // Enviamos el broadcast con el nombre de evento correcto
    a.Hub.Broadcast <- SocketMessage{
        Event: "vote_cast", 
        Payload: VoteUpdate{
            PollID:   input.PollID,
            OptionID: input.OptionID,
            NewCount: newCount,
        },
    }

    return nil, nil
}