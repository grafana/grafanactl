package resources

import (
	"context"

	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/grafana/grafanactl/internal/config"
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

	g, ctx := errgroup.WithContext(ctx)
	cmdRes := make([][]unstructured.Unstructured, len(resources))

	for i, r := range resources {
		g.Go(func() error {
			res, err := p.client.List(ctx, r, metav1.ListOptions{})
			if err == nil {
				cmdRes[i] = res.Items
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	res := make([]unstructured.Unstructured, 0, len(resources))
	for _, r := range cmdRes {
		res = append(res, r...)
	}

	return res, nil
}

func (p *Puller) Pull(ctx context.Context, cmd PullerCommand) ([]unstructured.Unstructured, error) {
	cmds, err := ParsePullCommands(cmd.Commands)
	if err != nil {
		return nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	cmdRes := make([][]unstructured.Unstructured, len(cmds))

	for i, c := range cmds {
		g.Go(func() error {
			switch c.Kind {
			case PullCommandTypeAll:
				res, err := p.client.List(ctx, c.GVK, metav1.ListOptions{})
				if err != nil {
					if !cmd.ContinueOnError {
						return err
					}
				} else {
					cmdRes[i] = res.Items
				}
			case PullCommandTypeMultiple:
				res, err := p.client.GetMultiple(ctx, c.GVK, c.UIDs, metav1.ListOptions{})
				if err != nil {
					if !cmd.ContinueOnError {
						return err
					}
				} else {
					cmdRes[i] = res
				}
			case PullCommandTypeSingle:
				res, err := p.client.Get(ctx, c.GVK, c.UIDs[0], metav1.GetOptions{})
				if err != nil {
					if !cmd.ContinueOnError {
						return err
					}
				} else {
					cmdRes[i] = []unstructured.Unstructured{*res}
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	res := make([]unstructured.Unstructured, 0, len(cmds))
	for _, r := range cmdRes {
		res = append(res, r...)
	}

	return res, nil
}
