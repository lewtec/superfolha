/**
 * @generated SignedSource<<07f3d6430ce40606a3c5033ef1d4d512>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type CommitProjectMutation$variables = {
  message: string;
  projectId: string;
};
export type CommitProjectMutation$data = {
  readonly commit: {
    readonly author: string;
    readonly date: any;
    readonly hash: string;
    readonly message: string;
  };
};
export type CommitProjectMutation = {
  response: CommitProjectMutation$data;
  variables: CommitProjectMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = {
  "defaultValue": null,
  "kind": "LocalArgument",
  "name": "message"
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
        "name": "message",
        "variableName": "message"
      },
      {
        "kind": "Variable",
        "name": "projectId",
        "variableName": "projectId"
      }
    ],
    "concreteType": "Commit",
    "kind": "LinkedField",
    "name": "commit",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "hash",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "message",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "author",
        "storageKey": null
      },
      {
        "alias": null,
        "args": null,
        "kind": "ScalarField",
        "name": "date",
        "storageKey": null
      }
    ],
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
    "name": "CommitProjectMutation",
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
    "name": "CommitProjectMutation",
    "selections": (v2/*: any*/)
  },
  "params": {
    "cacheID": "6685b2ced7d9df2d0548a6824726cdd1",
    "id": null,
    "metadata": {},
    "name": "CommitProjectMutation",
    "operationKind": "mutation",
    "text": "mutation CommitProjectMutation(\n  $projectId: ID!\n  $message: String!\n) {\n  commit(projectId: $projectId, message: $message) {\n    hash\n    message\n    author\n    date\n  }\n}\n"
  }
};
})();

(node as any).hash = "f1bcc098ce533e78775a4626896f7370";

export default node;
