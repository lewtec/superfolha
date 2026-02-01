/**
 * @generated SignedSource<<f05ecf173c5473874d0a7b345f99d8f4>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type DeleteFileMutation$variables = {
  path: string;
  projectId: string;
};
export type DeleteFileMutation$data = {
  readonly deleteFile: boolean;
};
export type DeleteFileMutation = {
  response: DeleteFileMutation$data;
  variables: DeleteFileMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "path"
},
v1 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "projectId"
},
v2 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "path",
        "variableName": "path"
      },
      {
        "kind": "Variable",
        "name": "projectId",
        "variableName": "projectId"
      }
    ],
    "kind": "ScalarField",
    "name": "deleteFile",
    "storageKey": null
  }
];
return {
  "fragment": {
    "argumentDefinitions": [
      (v0/*: any*/),
      (v1/*: any*/)
    ],
    "kind": "Fragment",
    "metadata": null,
    "name": "DeleteFileMutation",
    "selections": (v2/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [
      (v1/*: any*/),
      (v0/*: any*/)
    ],
    "kind": "Operation",
    "name": "DeleteFileMutation",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "b9b1d8c5155507b19857912118bbec15",
    "id": null,
    "metadata": {},
    "name": "DeleteFileMutation",
    "operationKind": "mutation",
    "text": "mutation DeleteFileMutation(\n  $projectId: ID!\n  $path: String!\n) {\n  deleteFile(projectId: $projectId, path: $path)\n}\n"
  }
};
})();

(node as any).hash = "c4400266278a4ecaf4767b9cf1819f18";

export default node;
