package models

import (
	"api_voty/ent"
	"api_voty/ent/device"
	"context"
	"time"
)

type DeviceModel struct {
	client *ent.Client
}

func NewDeviceModel(client *ent.Client) *DeviceModel {
	return &DeviceModel{client: client}
}

func (m *DeviceModel) UpdateToken(ctx context.Context, userID string, token string) error {
	// 1. Buscamos si el token ya está registrado
	id, err := m.client.Device.Query().
		Where(device.Token(token)).
		OnlyID(ctx)

	if err != nil {
		// 2. Si no existe, lo creamos
		if ent.IsNotFound(err) {
			return m.client.Device.Create().
				SetToken(token).
				SetUserID(userID).
				Exec(ctx)
		}
		return err
	}

	// 3. Si ya existe, actualizamos el usuario asociado y la fecha
	return m.client.Device.UpdateOneID(id).
		SetUserID(userID).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
}

func (m *DeviceModel) GetAllTokens(ctx context.Context) ([]string, error) {
	return m.client.Device.Query().Select(device.FieldToken).Strings(ctx)
}

func (m *DeviceModel) DeleteToken(ctx context.Context, token string) error {
	_, err := m.client.Device.Delete().Where(device.Token(token)).Exec(ctx)
	return err
}
