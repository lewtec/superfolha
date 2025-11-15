/**
 * @generated SignedSource<<6e51401fcef9bcc1ebb3cfde84ac77fa>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type GetProjectQuery$variables = {
  id: string;
};
export type GetProjectQuery$data = {
  readonly project: {
    readonly id: string;
    readonly name: string;
  } | null | undefined;
};
export type GetProjectQuery = {
  response: GetProjectQuery$data;
  variables: GetProjectQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "id"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "id",
        "variableName": "id"
      }
    ],
    "concreteType": "Project",
    "kind": "LinkedField",
    "name": "project",
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
    "name": "GetProjectQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GetProjectQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "d4a795c593588aad1587244e37809323",
    "id": null,
    "metadata": {},
    "name": "GetProjectQuery",
    "operationKind": "query",
    "text": "query GetProjectQuery(\n  $id: ID!\n) {\n  project(id: $id) {\n    id\n    name\n  }\n}\n"
  }
};
})();

(node as any).hash = "1dba6bdeef383b99296125783c8491c1";

export default node;
