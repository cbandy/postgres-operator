package kubeapi

/*
 Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"encoding/json"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

func PatchpgpolicyStatus(restclient *rest.RESTClient, state crv1.PgpolicyState, message string, oldCrd *crv1.Pgpolicy, namespace string) error {

	oldData, err := json.Marshal(oldCrd)
	if err != nil {
		return err
	}

	//change it
	oldCrd.Status = crv1.PgpolicyStatus{
		State:   state,
		Message: message,
	}

	//create the patch
	var newData, patchBytes []byte
	newData, err = json.Marshal(oldCrd)
	if err != nil {
		return err
	}
	patchBytes, err = jsonpatch.CreateMergePatch(oldData, newData)
	if err != nil {
		return err
	}

	log.Debug(string(patchBytes))

	//apply patch
	_, err6 := restclient.Patch(types.MergePatchType).
		Namespace(namespace).
		Resource(crv1.PgpolicyResourcePlural).
		Name(oldCrd.Spec.Name).
		Body(patchBytes).
		Do().
		Get()

	return err6

}
