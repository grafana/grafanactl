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

// PullerRequest describes a list of "pull" commands to get resources from Grafana.
type PullerRequest struct {
	Commands        []PullCommand
	ContinueOnError bool
}

// PullAll pulls all resources from Grafana.
func (p *Puller) PullAll(ctx context.Context) (*unstructured.UnstructuredList, error) {
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

			// TODO: honor "continue on error" flag + report error
			// return err
			return nil
		})
	}

	if err := errg.Wait(); err != nil {
		return nil, err
	}

	results := &unstructured.UnstructuredList{}
	results.NewEmptyInstance()
	results.SetKind("List")

	for _, r := range cmdRes {
		results.Items = append(results.Items, r...)
	}

	return results, nil
}

func (p *Puller) Pull(ctx context.Context, request PullerRequest) (*unstructured.UnstructuredList, error) {
	errg, ctx := errgroup.WithContext(ctx)
	cmdRes := make([][]unstructured.Unstructured, len(request.Commands))

	for idx, cmd := range request.Commands {
		errg.Go(func() error {
			switch cmd.Kind {
			case PullCommandTypeAll:
				res, err := p.client.List(ctx, cmd.GVK, metav1.ListOptions{})
				if err != nil {
					if !request.ContinueOnError {
						return err
					}
					// TODO: log error
				} else {
					cmdRes[idx] = res.Items
				}
			case PullCommandTypeMultiple:
				res, err := p.client.GetMultiple(ctx, cmd.GVK, cmd.UIDs, metav1.ListOptions{})
				if err != nil {
					if !request.ContinueOnError {
						return err
					}
					// TODO: log error
				} else {
					cmdRes[idx] = res
				}
			case PullCommandTypeSingle:
				res, err := p.client.Get(ctx, cmd.GVK, cmd.UIDs[0], metav1.GetOptions{})
				if err != nil {
					if !request.ContinueOnError {
						return err
					}
					// TODO: log error
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

	results := &unstructured.UnstructuredList{}
	results.SetKind("List")

	for _, r := range cmdRes {
		results.Items = append(results.Items, r...)
	}

	return results, nil
}
