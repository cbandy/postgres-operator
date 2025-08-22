# Copyright 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# This filter modifies a Kubernetes OpenAPI schema to help Kustomize apply patches.
# It reduces the schema to only the definitions for these GVKs:
#
# - CustomResourceDefinition.v1:
#   https://docs.k8s.io/reference/kubernetes-api/extend-resources/custom-resource-definition-v1
#

# The ".spec.versions" field is an atomic list, but we want to reach inside it using the "name" of each item.
(
  .definitions["io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.CustomResourceDefinitionSpec"]
    .properties.versions += {
      "x-kubernetes-list-type": "map",
      "x-kubernetes-list-map-keys": ["name"],
      "x-kubernetes-patch-merge-key": "name",
      "x-kubernetes-patch-strategy": "merge",
    }
) |

# TODO: "io.k8s.apiextensions-apiserver.pkg.apis.apiextensions.v1.JSONSchemaProps"

.
