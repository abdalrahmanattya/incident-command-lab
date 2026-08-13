package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/abdalrahmanattya/incident-command-lab/internal/store"
	"github.com/nats-io/nats.go"
	"time"
)

type Bus struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

func Connect(url string) (*Bus, error) {
	c, err := nats.Connect(url, nats.Timeout(3*time.Second), nats.MaxReconnects(5))
	if err != nil {
		return nil, err
	}
	js, err := c.JetStream()
	if err != nil {
		c.Close()
		return nil, err
	}
	_, err = js.AddStream(&nats.StreamConfig{Name: "INCIDENT_EVENTS", Subjects: []string{"incident.events.>"}, Storage: nats.FileStorage, MaxMsgs: -1})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		c.Close()
		return nil, err
	}
	_, err = js.AddStream(&nats.StreamConfig{Name: "INCIDENT_DLQ", Subjects: []string{"incident.dlq.>"}, Storage: nats.FileStorage, MaxMsgs: -1})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		c.Close()
		return nil, err
	}
	return &Bus{conn: c, js: js}, nil
}
func (b *Bus) PublishDLQ(ctx context.Context, subject string, data []byte) error {
	_, err := b.js.PublishMsg(&nats.Msg{Subject: "incident.dlq." + subject, Data: data}, nats.Context(ctx))
	return err
}
func (b *Bus) DLQInfo() (*nats.StreamInfo, error) { return b.js.StreamInfo("INCIDENT_DLQ") }
func (b *Bus) Close()                             { b.conn.Close() }
func (b *Bus) Publish(ctx context.Context, e store.Event) error {
	payload, _ := json.Marshal(e)
	_, err := b.js.PublishMsg(&nats.Msg{Subject: "incident.events." + e.Type, Data: payload}, nats.Context(ctx))
	return err
}
func (b *Bus) EnsureConsumer() error {
	_, err := b.js.AddConsumer("INCIDENT_EVENTS", &nats.ConsumerConfig{Durable: "incident-workers", AckPolicy: nats.AckExplicitPolicy, MaxDeliver: 5, FilterSubject: "incident.events.>"})
	if err != nil && err != nats.ErrConsumerNameAlreadyInUse {
		return fmt.Errorf("consumer: %w", err)
	}
	return nil
}
func (b *Bus) StartWorker(ctx context.Context, handler func(context.Context, *nats.Msg) error) {
	sub, err := b.js.PullSubscribe("incident.events.>", "incident-workers", nats.BindStream("INCIDENT_EVENTS"))
	if err != nil {
		return
	}
	go func() {
		for {
			msgs, err := sub.Fetch(10, nats.MaxWait(500*time.Millisecond))
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			for _, m := range msgs {
				if err := handler(ctx, m); err != nil {
					if md, e := m.Metadata(); e == nil && md.NumDelivered >= 5 {
						if e := b.PublishDLQ(ctx, m.Subject, m.Data); e != nil {
							fmt.Printf("dlq publish failed: %v\n", e)
							_ = m.NakWithDelay(time.Second)
						} else {
							_ = m.Ack()
						}
					} else {
						_ = m.NakWithDelay(time.Second)
					}
				} else {
					_ = m.Ack()
				}
			}
		}
	}()
}
func (b *Bus) StartRoleWorker(ctx context.Context, role string, subject string, handler func(context.Context, *nats.Msg) error) error {
	durable := "worker-" + role
	_, err := b.js.AddConsumer("INCIDENT_EVENTS", &nats.ConsumerConfig{Durable: durable, AckPolicy: nats.AckExplicitPolicy, MaxDeliver: 5, FilterSubject: subject})
	if err != nil && err != nats.ErrConsumerNameAlreadyInUse {
		return err
	}
	sub, err := b.js.PullSubscribe(subject, durable, nats.BindStream("INCIDENT_EVENTS"))
	if err != nil {
		return err
	}
	go func() {
		for {
			msgs, e := sub.Fetch(10, nats.MaxWait(500*time.Millisecond))
			if e != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			for _, m := range msgs {
				if e := handler(ctx, m); e != nil {
					if md, x := m.Metadata(); x == nil && md.NumDelivered >= 5 {
						if e := b.PublishDLQ(ctx, subject, m.Data); e != nil {
							fmt.Printf("dlq publish failed: %v\n", e)
							_ = m.NakWithDelay(time.Second)
						} else {
							_ = m.Ack()
						}
					} else {
						_ = m.NakWithDelay(time.Second)
					}
				} else {
					_ = m.Ack()
				}
			}
		}
	}()
	return nil
}
