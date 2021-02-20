package gofman

import (
  "context"
)

// Actor constants.
const (
  MaxActorNameLen = 255
)

// Actor represents an actor in the system.
type Actor struct {
  ID        string `json:"id"`
  UserID    string `json:"users_id"`
  Name      string `json:"name"`
  CreatedAt int64  `json:"created_at"`
  UpdatedAt int64  `json:"updated_at"`
  RemovedAt int64  `json:"removed_at"`
}

// Validate returns an error if the actor contains invalid fields.
func (t *Actor) Validate() error {
  if t.UserID == "" {
    return NewError(EINVALID, "User ID required.")
  }

  if t.Name == "" {
    return NewError(EINVALID, "Name required.")
  }

  if len(t.Name) > MaxActorNameLen {
    return NewError(EINVALID, "Name must be less than %d characters.", MaxActorNameLen)
  }

  return nil
}

// CanFindActor returns true if the current user can list actors with
// the given filter.
func CanFindActor(ctx context.Context, filter ActorFilter) bool {
  id := UserIDFromContext(ctx)
  return id != "" && filter.UserID == &id
}

// CanUpdateActor returns true if the current user can update the actor.
func CanUpdateActor(ctx context.Context, actor *Actor) bool {
  if user := UserFromContext(ctx); user != nil && user.IsDemo {
    return false
  } else {
    id := UserIDFromContext(ctx)
    return id != "" && actor.UserID == id
  }
}

// ActorService represents a service for managing actors. The functions
// should return ENOTFOUND if the actor could not be found and EUNAUTHORIZED
// if the user is not authorized to run the transaction.
type ActorService interface {
  FindActorByID(ctx context.Context, id string) (*Actor, error)
  FindActors(ctx context.Context, filter ActorFilter) ([]*Actor, int, error)
  CreateActor(ctx context.Context, actor *Actor) error
  UpdateActor(ctx context.Context, id string, update ActorUpdate) (*Actor, error)
  RemoveActor(ctx context.Context, id string) error
}

// ActorFilter represents a filter passed to FindActors().
type ActorFilter struct {
  ID     *string `json:"id"`
  UserID *string `json:"users_id"`

  Offset int `json:"offset"`
  Limit  int `json:"limit"`
}

// ActorUpdate represents a set of fields to be updated via UpdateActor().
type ActorUpdate struct {
  Name *string `json:"name"`
}
