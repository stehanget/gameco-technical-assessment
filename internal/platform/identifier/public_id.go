package identifier

import "github.com/google/uuid"

func NewPublicID() (uuid.UUID, error) {
	return uuid.NewV7()
}
