/**
 * @generated SignedSource<<61e5000b2b429becfa799333ef39728b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type RegisterMutation$variables = {
  email: string;
  password: string;
};
export type RegisterMutation$data = {
  readonly register: {
    readonly user: {
      readonly email: string;
      readonly id: string;
    };
  };
};
export type RegisterMutation = {
  response: RegisterMutation$data;
  variables: RegisterMutation$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "email"
  },
  {
    "defaultValue": null,
    "kind": "LocalArgument",
    "name": "password"
  }
],
v1 = [
  {
    "alias": null,
    "args": [
      {
        "kind": "Variable",
        "name": "email",
        "variableName": "email"
      },
      {
        "kind": "Variable",
        "name": "password",
        "variableName": "password"
      }
    ],
    "concreteType": "AuthPayload",
    "kind": "LinkedField",
    "name": "register",
    "plural": false,
    "selections": [
      {
        "alias": null,
        "args": null,
        "concreteType": "User",
        "kind": "LinkedField",
        "name": "user",
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
            "name": "email",
            "storageKey": null
          }
        ],
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
    "name": "RegisterMutation",
    "selections": (v1/*: any*/),
    "type": "Mutation",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": (v0/*: any*/),
    "kind": "Operation",
    "name": "RegisterMutation",
    "selections": (v1/*: any*/)
  },
  "params": {
    "cacheID": "cbe79788dae3073ce7580f3b2b5e1770",
    "id": null,
    "metadata": {},
    "name": "RegisterMutation",
    "operationKind": "mutation",
    "text": "mutation RegisterMutation(\n  $email: String!\n  $password: String!\n) {\n  register(email: $email, password: $password) {\n    user {\n      id\n      email\n    }\n  }\n}\n"
  }
};
})();

(node as any).hash = "60d53ac8fb8a3748304079d3b5a0371b";

export default node;
