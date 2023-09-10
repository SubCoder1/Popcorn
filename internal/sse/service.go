// Service layer of Server Side Events (SSE) in Popcorn.

package sse

import (
	"Popcorn/internal/entity"
	"Popcorn/pkg/log"
	"context"
	"sync"
	"time"
)

type Service interface {
	// Initializes and / or returns an instance of entity.SSE
	GetOrSetEvent(ctx context.Context) *entity.SSE
	// Launch a listener for SSE, preferably in a goroutine for non-blockage
	Listen(ctx context.Context)
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	logger log.Logger
}

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(logger log.Logger) Service {
	return service{logger}
}

// Global Instance of entity.SSE initialized via GetOrSetEvent().
var event *entity.SSE

// Quit signal to force close SSE channels before server shutdown
var quit chan bool

// sync.Once singleton is used to make sure event instantiation is done only once.
var once sync.Once

func (s service) GetOrSetEvent(ctx context.Context) *entity.SSE {
	once.Do(func() {
		quit = make(chan bool)
		event = &entity.SSE{
			Message:       make(chan entity.SSEData),
			NewClients:    make(chan entity.SSEClient),
			ClosedClients: make(chan entity.SSEClient),
			TotalClients:  make(map[string]chan entity.SSEData),
		}
		s.logger.WithCtx(ctx).Info().Msg("Initialized Popcorn SSE instance.")
	})
	return event
}

func (s service) Listen(ctx context.Context) {
	for {
		select {
		// Add new available client
		case client, ok := <-s.GetOrSetEvent(ctx).NewClients:
			if !ok {
				s.logger.WithCtx(ctx).Error().Msgf("Error occured while setting new SSE channel for %s", client.ID)
			} else {
				s.GetOrSetEvent(ctx).TotalClients[client.ID] = client.Channel
				s.logger.WithCtx(ctx).Info().Msgf("Added client %s into Popcorn SSE event channel", client.ID)
			}

		// Remove closed client
		case client, ok := <-s.GetOrSetEvent(ctx).ClosedClients:
			if ok {
				close(client.Channel)
				delete(s.GetOrSetEvent(ctx).TotalClients, client.ID)
				s.logger.WithCtx(ctx).Info().Msgf("Removed client %s from Popcorn SSE event channel", client.ID)
			}

		// Broadcast message to a specific client with client ID fetched from eventMsg.To
		case eventMsg, ok := <-s.GetOrSetEvent(ctx).Message:
			if ok && s.GetOrSetEvent(ctx).TotalClients[eventMsg.To] != nil {
				s.GetOrSetEvent(ctx).TotalClients[eventMsg.To] <- eventMsg
			}
		}
	}
}

func Cleanup(ctx context.Context) error {
	// This quit signal will close open stream API connections
	close(quit)
	go func() {
		time.Sleep(1 * time.Second)
		close(event.Message)
		close(event.ClosedClients)
		close(event.NewClients)
	}()
	return nil
}
