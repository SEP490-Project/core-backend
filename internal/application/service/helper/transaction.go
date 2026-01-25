package helper

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
)

func EnsureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func WithTransaction(
	ctx context.Context,
	uow irepository.UnitOfWork,
	fn func(ctx context.Context, uow irepository.UnitOfWork) error,
) (err error) {
	ctx = EnsureContext(ctx)

	startedHere := false
	if !uow.InTransaction() {
		uow = uow.Begin(ctx)
		startedHere = true
	}

	defer func() {
		if r := recover(); r != nil {
			_ = uow.Rollback()
			panic(r)
		}
		if startedHere {
			if err == nil {
				err = uow.Commit()
			} else {
				_ = uow.Rollback()
			}
		}
	}()

	err = fn(ctx, uow)
	return err
}
