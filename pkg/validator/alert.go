package validator

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fusakla/promruval/v2/pkg/prometheus"
	"github.com/fusakla/promruval/v2/pkg/unmarshaler"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/template"
	"gopkg.in/yaml.v3"
)

func newForIsNotLongerThan(paramsConfig yaml.Node) (Validator, error) {
	params := struct {
		Limit model.Duration `yaml:"limit"`
	}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	if params.Limit == model.Duration(0) {
		return nil, fmt.Errorf("missing limit")
	}
	return &forIsNotLongerThan{limit: params.Limit}, nil
}

type forIsNotLongerThan struct {
	limit model.Duration
}

func (h forIsNotLongerThan) String() string {
	return fmt.Sprintf("`for` is not longer than `%s`", h.limit)
}

func (h forIsNotLongerThan) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	if rule.For != 0 && rule.For > h.limit {
		return []error{fmt.Errorf("alert has `for: %s` which is longer than the specified limit of %s", rule.For, h.limit)}
	}
	return nil
}

func newValidateLabelTemplates(paramsConfig yaml.Node) (Validator, error) {
	params := struct{}{}
	if err := paramsConfig.Decode(&params); err != nil {
		return nil, err
	}
	return &validateLabelTemplates{}, nil
}

type validateLabelTemplates struct{}

func (h validateLabelTemplates) String() string {
	return "labels are valid templates"
}

func (h validateLabelTemplates) Validate(_ unmarshaler.RuleGroup, rule rulefmt.Rule, _ *prometheus.Client) []error {
	var errs []error
	data := template.AlertTemplateData(map[string]string{}, map[string]string{}, "", 0)
	defs := []string{
		"{{$labels := .Labels}}",
		"{{$externalLabels := .ExternalLabels}}",
		"{{$externalURL := .ExternalURL}}",
		"{{$value := .Value}}",
	}
	for k, v := range rule.Labels {
		t := template.NewTemplateExpander(context.TODO(), strings.Join(append(defs, v), ""), k, data, model.Now(), func(_ context.Context, _ string, _ time.Time) (promql.Vector, error) { return nil, nil }, &url.URL{}, []string{})
		if _, err := t.Expand(); err != nil && !strings.Contains(err.Error(), "error executing template") {
			errs = append(errs, fmt.Errorf("invalid template of label %s: %w", k, err))
		}
	}
	return errs
}
