package action

import (
	"github.com/bryanl/woowoo/env"
	"github.com/bryanl/woowoo/pkg/client"
	"github.com/spf13/afero"
)

// Delete deletes from an environment.
func Delete(fs afero.Fs, env string, options client.DeleteOptions, opts ...DeleteOpt) error {
	s, err := newDelete(fs, env, options, opts...)
	if err != nil {
		return err
	}

	return s.Run()
}

// DeleteOpt is an option for configuring Delete.
type DeleteOpt func(*delete)

// DeleteWithComponents selects the components to be delete.
func DeleteWithComponents(names ...string) DeleteOpt {
	return func(s *delete) {
		s.components = names
	}
}

// Delete is a delete Action
type delete struct {
	env        string
	components []string
	options    client.DeleteOptions

	*base
}

// NewDelete creates an instance of Delete.
func newDelete(fs afero.Fs, env string, options client.DeleteOptions, opts ...DeleteOpt) (*delete, error) {
	b, err := new(fs)
	if err != nil {
		return nil, err
	}

	s := &delete{
		env:     env,
		options: options,
		base:    b,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// Run runs the action.
func (s *delete) Run() error {
	return env.Delete(s.app, s.env, s.components, s.options)
}