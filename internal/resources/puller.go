package resources

import (
	"context"

	"github.com/grafana/grafanactl/internal/config"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Puller is a command that pulls resources from Grafana.
type Puller struct {
	client *NamespacedDynamicClient
}

// NewPuller creates a new Puller.
func NewPuller(cfg config.Context) (*Puller, error) {
	rcfg, err := config.NewNamespacedRESTConfig(cfg)
	if err != nil {
		return nil, err
	}

	client, err := NewDefaultNamespacedDynamicClient(rcfg)
	if err != nil {
		return nil, err
	}

	return &Puller{client: client}, nil
}

// PullerCommand is a command that pulls resources from Grafana.
type PullerCommand struct {
	Commands        []string
	ContinueOnError bool
}

// PullAll pulls all resources from Grafana.
func (p *Puller) PullAll(ctx context.Context) ([]unstructured.Unstructured, error) {
	resources, err := p.client.Resources(ctx)
	if err != nil {
		return nil, err
	}

	errg, ctx := errgroup.WithContext(ctx)
	cmdRes := make([][]unstructured.Unstructured, len(resources))

	for i, r := range resources {
		errg.Go(func() error {
			res, err := p.client.List(ctx, r, metav1.ListOptions{})
			if err == nil {
				cmdRes[i] = res.Items
			}

			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return nil, err
	}

	res := make([]unstructured.Unstructured, 0, len(resources))
	for _, r := range cmdRes {
		res = append(res, r...)
	}

	return res, nil
}

func (p *Puller) Pull(ctx context.Context, command PullerCommand) ([]unstructured.Unstructured, error) {
	cmds, err := ParsePullCommands(command.Commands)
	if err != nil {
		return nil, err
	}

	errg, ctx := errgroup.WithContext(ctx)
	cmdRes := make([][]unstructured.Unstructured, len(cmds))

	for idx, cmd := range cmds {
		errg.Go(func() error {
			switch cmd.Kind {
			case PullCommandTypeAll:
				res, err := p.client.List(ctx, cmd.GVK, metav1.ListOptions{})
				if err != nil {
					if !command.ContinueOnError {
						return err
					}
				} else {
					cmdRes[idx] = res.Items
				}
			case PullCommandTypeMultiple:
				res, err := p.client.GetMultiple(ctx, cmd.GVK, cmd.UIDs, metav1.ListOptions{})
				if err != nil {
					if !command.ContinueOnError {
						return err
					}
				} else {
					cmdRes[idx] = res
				}
			case PullCommandTypeSingle:
				res, err := p.client.Get(ctx, cmd.GVK, cmd.UIDs[0], metav1.GetOptions{})
				if err != nil {
					if !command.ContinueOnError {
						return err
					}
				} else {
					cmdRes[idx] = []unstructured.Unstructured{*res}
				}
			}

			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return nil, err
	}

	res := make([]unstructured.Unstructured, 0, len(cmds))
	for _, r := range cmdRes {
		res = append(res, r...)
	}

	return res, nil
}
