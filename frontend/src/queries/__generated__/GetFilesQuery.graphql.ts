/**
 * @generated SignedSource<<fe912e0234d249c2e82989029be22b91>>
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
    readonly content: string | null | undefined;
    readonly isTooBig: boolean;
    readonly path: string;
    readonly size: number;
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
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "size",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "isTooBig",
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
    "cacheID": "bfebf232c0a5530ea43cefd3cad42c22",
    "id": null,
    "metadata": {},
    "name": "GetFilesQuery",
    "operationKind": "query",
    "text": "query GetFilesQuery(\n  $projectId: ID!\n) {\n  files(projectId: $projectId) {\n    path\n    content\n    size\n    isTooBig\n  }\n}\n"
  }
};
})();

(node as any).hash = "3c4aad4ac9c4405b933114850aef45d9";

export default node;
