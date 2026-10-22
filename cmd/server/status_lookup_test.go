package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/msgfy/linktor/internal/domain/entity"
)

type finderStub struct {
	calls  int
	readyAt int
}

func (f *finderStub) FindByExternalID(_ context.Context, id string) (*entity.Message, error) {
	f.calls++
	if f.calls >= f.readyAt {
		return &entity.Message{ID: "m-" + id, ExternalID: id}, nil
	}
	return nil, errors.New("not found")
}

// O recibo que chega antes do ack gravar o external_id encontra a mensagem na
// tentativa seguinte; o que nunca aparece desiste depois do limite, sem travar.
func TestFindMessageByExternalID_RetriesBriefly(t *testing.T) {
	var waited time.Duration
	sleep := func(d time.Duration) { waited += d }

	late := &finderStub{readyAt: 3}
	msg, err := findMessageByExternalID(context.Background(), late, "WA1", sleep)
	if err != nil || msg == nil || msg.ID != "m-WA1" {
		t.Fatalf("esperava achar na 3ª tentativa: msg=%v err=%v", msg, err)
	}
	if late.calls != 3 || waited != 2*externalIDLookupDelay {
		t.Fatalf("calls=%d waited=%s", late.calls, waited)
	}

	never := &finderStub{readyAt: 100}
	if _, err := findMessageByExternalID(context.Background(), never, "WA2", sleep); err == nil {
		t.Fatal("esperava erro quando a mensagem não existe")
	}
	if never.calls != externalIDLookupRetries {
		t.Fatalf("calls=%d, esperava %d", never.calls, externalIDLookupRetries)
	}
}
