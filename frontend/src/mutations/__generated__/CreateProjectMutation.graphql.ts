/**
 * @generated SignedSource<<eb2b26d70de7f81a0d6d565960c2be0f>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type CreateProjectMutation$variables = {
  name: string;
};
export type CreateProjectMutation$data = {
  readonly createProject: {
    readonly createdAt: any;
    readonly id: string;
    readonly name: string;
    readonly updatedAt: any;
  };
};
export type CreateProjectMutation = {
  response: CreateProjectMutation$data;
  variables: CreateProjectMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "name"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "name",
        "variableName": "name"
      }
    ],
    "concreteType": "Project",
    "kind": "LinkedField",
    "name": "createProject",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "id",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "name",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "createdAt",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "updatedAt",
        "storageKey": null
      }
    ],
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Fragment",
    "metadata": null,
    "name": "CreateProjectMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "CreateProjectMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "7aec209cf140686aec197b08066db1ca",
    "id": null,
    "metadata": {},
    "name": "CreateProjectMutation",
    "operationKind": "mutation",
    "text": "mutation CreateProjectMutation(\n  $name: String!\n) {\n  createProject(name: $name) {\n    id\n    name\n    createdAt\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "73933ff9406c82b8ce4a1e91a61b54ef";

export default node;
