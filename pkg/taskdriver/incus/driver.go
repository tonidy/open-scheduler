package incus

import (
	"context"

	pb "github.com/open-scheduler/proto"
)

type IncusDriver struct {
	ctx context.Context
}

func NewIncusDriver() *IncusDriver {
	return &IncusDriver{
		ctx: context.Background(),
	}
}

func (d *IncusDriver) Run(ctx context.Context, task *pb.Task) error {
	// TODO: Implement Incus driver
	return nil
}
