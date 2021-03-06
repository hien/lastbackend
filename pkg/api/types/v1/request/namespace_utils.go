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

package request

import (
	"encoding/json"
	"github.com/lastbackend/lastbackend/pkg/distribution/errors"
	"github.com/lastbackend/lastbackend/pkg/util/validator"
	"io"
	"io/ioutil"
)

type NamespaceRequest struct{}

func (NamespaceRequest) Manifest() *NamespaceManifest {
	return new(NamespaceManifest)
}

func (s *NamespaceManifest) Validate() *errors.Err {
	switch true {
	case s.Meta.Name != nil && !validator.IsServiceName(*s.Meta.Name):
		return errors.New("namespace").BadParameter("name")
	case s.Meta.Description != nil && len(*s.Meta.Description) > DEFAULT_DESCRIPTION_LIMIT:
		return errors.New("namespace").BadParameter("description")
	}

	return nil
}

func (s *NamespaceManifest) DecodeAndValidate(reader io.Reader) *errors.Err {

	if reader == nil {
		err := errors.New("data body can not be null")
		return errors.New("namespace").IncorrectJSON(err)
	}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return errors.New("namespace").Unknown(err)
	}

	err = json.Unmarshal(body, s)
	if err != nil {
		return errors.New("namespace").IncorrectJSON(err)
	}

	if err := s.Validate(); err != nil {
		return err
	}

	return nil
}

func (NamespaceRequest) RemoveOptions() *NamespaceRemoveOptions {
	return new(NamespaceRemoveOptions)
}

func (n *NamespaceRemoveOptions) Validate() *errors.Err {
	return nil
}

func (n *NamespaceRemoveOptions) ToJson() ([]byte, error) {
	return json.Marshal(n)
}
