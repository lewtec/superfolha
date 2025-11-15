/**
 * @generated SignedSource<<4a9e63778a0632086ceb88f0eb26c251>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type GetFilesQuery$variables = {
  projectId: string;
};
export type GetFilesQuery$data = {
  readonly files: ReadonlyArray<{
    readonly content: string;
    readonly path: string;
  }>;
};
export type GetFilesQuery = {
  response: GetFilesQuery$data;
  variables: GetFilesQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "projectId"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "projectId",
        "variableName": "projectId"
      }
    ],
    "concreteType": "File",
    "kind": "LinkedField",
    "name": "files",
    "plural": true,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "path",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "content",
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
    "name": "GetFilesQuery",
    "selections": (v1/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "GetFilesQuery",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "96ea4988e7d9afe15eb6faf36761cb1e",
    "id": null,
    "metadata": {},
    "name": "GetFilesQuery",
    "operationKind": "query",
    "text": "query GetFilesQuery(\n  $projectId: ID!\n) {\n  files(projectId: $projectId) {\n    path\n    content\n  }\n}\n"
  }
};
})();

(node as any).hash = "5b27a552d660f68ed08aa0bbd13dde8d";

export default node;
