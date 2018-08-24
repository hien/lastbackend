//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2018] Last.Backend LLC
// All Rights Reserved.
//
// NOTICE:  All information contained herein is, and remains
// the property of Last.Backend LLC and its suppliers,
// if any.  The intellectual and technical concepts contained
// herein are proprietary to Last.Backend LLC
// and its suppliers and may be covered by Russian Federation and Foreign Patents,
// patents in process, and are protected by trade secret or copyright law.
// Dissemination of this information or reproduction of this material
// is strictly forbidden unless prior written permission is obtained
// from Last.Backend LLC.
//

package distribution

import (
	"context"
	"fmt"
	"strings"

	"encoding/json"

	"github.com/lastbackend/lastbackend/pkg/distribution/errors"
	"github.com/lastbackend/lastbackend/pkg/distribution/types"
	"github.com/lastbackend/lastbackend/pkg/log"
	"github.com/lastbackend/lastbackend/pkg/storage"
	"github.com/spf13/viper"
)

const (
	logServicePrefix = "distribution:service"
)

type Service struct {
	context context.Context
	storage storage.Storage
}

func (s *Service) Runtime() (*types.Runtime, error) {

	log.V(logLevel).Debugf("%s:get:> get services runtime info", logServicePrefix)
	runtime, err := s.storage.Info(s.context, s.storage.Collection().Service(), "")
	if err != nil {
		log.V(logLevel).Errorf("%s:get:> get runtime info error: %s", logServicePrefix, err)
		return &runtime.Runtime, err
	}
	return &runtime.Runtime, nil

}

// Get service by namespace and service name
func (s *Service) Get(namespace, name string) (*types.Service, error) {

	log.V(logLevel).Debugf("%s:get:> get in namespace %s by name %s", logServicePrefix, namespace, name)

	svc := new(types.Service)

	err := s.storage.Get(s.context, s.storage.Collection().Service(), s.storage.Key().Service(namespace, name), svc, nil)
	if err != nil {

		if errors.Storage().IsErrEntityNotFound(err) {
			log.V(logLevel).Warnf("%s:get:> get in namespace %s by name %s not found", logServicePrefix, namespace, name)
			return nil, nil
		}

		log.V(logLevel).Errorf("%s:get:> get in namespace %s by name %s error: %v", logServicePrefix, namespace, name, err)
		return nil, err
	}

	return svc, nil
}

// List method return map of services in selected namespace
func (s *Service) List(namespace string) (*types.ServiceList, error) {

	log.V(logLevel).Debugf("%s:list:> by namespace %s", logServicePrefix, namespace)

	list := types.NewServiceList()
	q := s.storage.Filter().Service().ByNamespace(namespace)

	err := s.storage.List(s.context, s.storage.Collection().Service(), q, list, nil)
	if err != nil {
		log.V(logLevel).Error("%s:list:> by namespace %s err: %v", logServicePrefix, namespace, err)
		return nil, err
	}

	log.V(logLevel).Debugf("%s:list:> by namespace %s result: %d", logServicePrefix, namespace, len(list.Items))

	return list, nil
}

// Create new service model in namespace
func (s *Service) Create(namespace *types.Namespace, opts *types.ServiceCreateOptions) (*types.Service, error) {

	log.V(logLevel).Debugf("%s:create:> create new service %#v", logServicePrefix, opts)

	service := new(types.Service)
	switch true {
	case opts == nil:
		return nil, errors.New("opts can not be nil")
	case opts.Name == nil || *opts.Name == "":
		return nil, errors.New("name is required")
	case opts.Image == nil:
		return nil, errors.New("image is required")
	case opts.Image.Name == nil  || *opts.Image.Name == types.EmptyString:
		return nil, errors.New("image is required")
	}

	// prepare meta data for service
	service.Meta.SetDefault()
	service.Meta.Name = *opts.Name
	service.Meta.Namespace = namespace.Meta.Name
	service.Meta.Endpoint = strings.ToLower(fmt.Sprintf("%s-%s.%s", *opts.Name, namespace.Meta.Name, viper.GetString("domain.internal")))

	if opts.Description != nil {
		service.Meta.Description = *opts.Description
	}

	service.SelfLink()

	service.Status.State = types.StateCreated
	// prepare default template spec
	c := types.SpecTemplateContainer{}
	c.SetDefault()
	c.Role = types.ContainerRolePrimary

	// prepare spec data for service
	service.Spec.SetDefault()
	service.Spec.Template.Containers = append(service.Spec.Template.Containers, c)

	if opts.Spec != nil {
		service.Spec.Update(opts.Image, opts.Spec)
	}

	service.SelfLink()

	if err := s.storage.Put(s.context, s.storage.Collection().Service(),
		s.storage.Key().Service(service.Meta.Namespace, service.Meta.Name), service, nil); err != nil {
		log.V(logLevel).Errorf("%s:create:> insert service err: %v", logServicePrefix, err)
		return nil, err
	}

	return service, nil
}

// Update service in namespace
func (s *Service) Update(service *types.Service, opts *types.ServiceUpdateOptions) (*types.Service, error) {

	log.V(logLevel).Debugf("%s:update:> %#v -> %#v", logServicePrefix, service, opts)

	if opts == nil {
		opts = new(types.ServiceUpdateOptions)
	}

	if opts.Description != nil {
		service.Meta.Description = *opts.Description
	}

	if opts.Spec != nil || opts.Image != nil {
		service.Status.State = types.StateProvision
		service.Spec.Update(opts.Image, opts.Spec)
	}

	if err := s.storage.Set(s.context, s.storage.Collection().Service(),
		s.storage.Key().Service(service.Meta.Namespace, service.Meta.Name), service, nil); err != nil {
		log.V(logLevel).Errorf("%s:update:> update service spec err: %v", logServicePrefix, err)
		return nil, err
	}

	return service, nil
}

// Destroy method marks service for destroy
func (s *Service) Destroy(service *types.Service) (*types.Service, error) {

	log.V(logLevel).Debugf("%s:destroy:> destroy service %s", logServicePrefix, service.SelfLink())

	service.Status.State = types.StateDestroy
	service.Spec.State.Destroy = true

	if err := s.storage.Set(s.context, s.storage.Collection().Service(),
		s.storage.Key().Service(service.Meta.Namespace, service.Meta.Name), service, nil); err != nil {
		log.V(logLevel).Errorf("%s:destroy:> destroy service err: %v", logServicePrefix, err)
		return nil, err
	}
	return service, nil
}

// Remove service from storage
func (s *Service) Remove(service *types.Service) error {

	log.V(logLevel).Debugf("%s:remove:> remove service %#v", logServicePrefix, service)

	err := s.storage.Del(s.context, s.storage.Collection().Service(),
		s.storage.Key().Service(service.Meta.Namespace, service.Meta.Name))
	if err != nil {
		log.V(logLevel).Errorf("%s:remove:> remove service err: %v", logServicePrefix, err)
		return err
	}

	return nil
}

// Set state for deployment
func (s *Service) Set(service *types.Service) error {

	if service == nil {
		return errors.New(errors.ErrStructArgIsNil)
	}


	log.V(logLevel).Debugf("%s:setstatus:> set state for service %s", logServicePrefix, service.Meta.Name)

	key := s.storage.Key().Service(service.Meta.Namespace, service.Meta.Name)
	if err := s.storage.Set(s.context, s.storage.Collection().Service(), key, service, nil); err != nil {
		log.Errorf("%s:setstatus:> set state for service %s err: %v", logServicePrefix, service.Meta.Name, err)
		return err
	}

	return nil
}

// Watch service changes
func (s *Service) Watch(ch chan types.ServiceEvent, rev *int64) error {

	log.V(logLevel).Debugf("%s:watch:> watch service by spec changes", logServicePrefix)

	done := make(chan bool)
	watcher := storage.NewWatcher()

	go func() {
		for {
			select {
			case <-s.context.Done():
				done <- true
				return
			case e := <-watcher:
				if e.Data == nil {
					continue
				}

				res := types.ServiceEvent{}
				res.Action = e.Action
				res.Name = e.Name

				service := new(types.Service)

				if err := json.Unmarshal(e.Data.([]byte), service); err != nil {
					log.Errorf("%s:> parse data err: %v", logServicePrefix, err)
					continue
				}

				res.Data = service

				ch <- res
			}
		}
	}()

	opts := storage.GetOpts()
	opts.Rev = rev
	if err := s.storage.Watch(s.context, s.storage.Collection().Service(), watcher, opts); err != nil {
		return err
	}

	return nil
}

// NewServiceModel returns new service management model
func NewServiceModel(ctx context.Context, stg storage.Storage) *Service {
	return &Service{ctx, stg}
}
