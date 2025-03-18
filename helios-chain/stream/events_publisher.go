package stream

import (
	"context"
	"fmt"
	"os"

	"helios-core/helios-chain/stream/types"

	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub"
	"github.com/cosmos/cosmos-sdk/baseapp"
)

type Topic string

const BankBalances = Topic("cosmos.bank.v1beta1.EventSetBalances")

const StreamEvents = "stream.events"

type eventHandler = func(buffer *types.StreamResponseMap, event abci.Event) error

type Publisher struct {
	inABCIEvents   chan baseapp.StreamEvents
	bus            *pubsub.Server
	done           chan struct{}
	eventHandlers  map[Topic]eventHandler
	bufferCapacity uint
}

func NewPublisher(inABCIEvents chan baseapp.StreamEvents, bus *pubsub.Server) *Publisher {
	p := &Publisher{
		inABCIEvents:   inABCIEvents,
		bus:            bus,
		done:           make(chan struct{}),
		eventHandlers:  make(map[Topic]eventHandler),
		bufferCapacity: 100,
	}
	p.registerHandlers()
	return p
}

func (e *Publisher) Run(ctx context.Context) error {
	logger := log.NewLogger(os.Stderr)
	err := e.bus.Start()
	if err != nil {
		return fmt.Errorf("failed to start pubsub server: %w", err)
	}

	eventsBuffer := make(chan baseapp.StreamEvents, e.bufferCapacity)

	go func() {
		for {
			events := <-e.inABCIEvents
			select {
			case eventsBuffer <- events:
			default:
				if e.bus.IsRunning() {
					logger.Error("eventsBuffer is full, chain streamer will be stopped")
					if err = e.bus.Publish(ctx, fmt.Errorf("chain stream event buffer overflow")); err != nil {
						logger.Error("failed to publish", "error", err)
					}
					err = e.Stop()
					if err != nil {
						logger.Error("failed to stop event publisher", "error", err)
					}
				}
			}
		}
	}()

	go func() {
		inBuffer := types.NewStreamResponseMap()
		for {
			select {
			case <-e.done:
				return
			case events := <-eventsBuffer:
				// The block height is required in the inBuffer when calculating the id for trade events
				inBuffer.BlockHeight = events.Height

				for _, ev := range events.Events {
					if handler, ok := e.eventHandlers[Topic(ev.Type)]; ok {
						err := handler(inBuffer, ev)
						if err != nil {
							if he := e.bus.Publish(ctx, err); he != nil {
								logger.Error("failed to publish", "error", err)
							}
						}
					}
				}

				// all events for specific height are received
				if events.Flush {
					inBuffer.BlockHeight = events.Height
					inBuffer.BlockTime = events.BlockTime
					// flush buffer
					if err := e.bus.Publish(ctx, inBuffer); err != nil {
						logger.Error("failed to publish stream response", "error", err)
					}
					// clear buffer
					inBuffer = types.NewStreamResponseMap()
				}
			}
		}
	}()

	return nil
}

func (e *Publisher) Stop() error {
	if !e.bus.IsRunning() {
		return nil
	}
	log.NewLogger(os.Stderr).Info("stopping stream publisher")
	err := e.bus.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop pubsub server: %w", err)
	}
	e.done <- struct{}{}
	return nil
}

func (e *Publisher) registerHandlers() {
	// Register events
	e.RegisterEventHandler(BankBalances, handleBankBalanceEvent)
}

func (e *Publisher) RegisterEventHandler(topic Topic, handler eventHandler) {
	e.eventHandlers[topic] = handler
}

func (e *Publisher) WithBufferCapacity(capacity uint) *Publisher {
	e.bufferCapacity = capacity
	return e
}
