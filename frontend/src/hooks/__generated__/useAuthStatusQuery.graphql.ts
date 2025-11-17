/**
 * @generated SignedSource<<21be06f386cbb899f4406a510c2bd53a>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type useAuthStatusQuery$variables = Record<PropertyKey, never>;
export type useAuthStatusQuery$data = {
  readonly me: {
    readonly email: string;
    readonly id: string;
  } | null | undefined;
};
export type useAuthStatusQuery = {
  response: useAuthStatusQuery$data;
  variables: useAuthStatusQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "concreteType": "User",
    "kind": "LinkedField",
    "name": "me",
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
];
return {
  "fragment": {
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "useAuthStatusQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "useAuthStatusQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "b07113c8d4796cdd9bb24bea63a117fd",
    "id": null,
    "metadata": {},
    "name": "useAuthStatusQuery",
    "operationKind": "query",
    "text": "query useAuthStatusQuery {\n  me {\n    id\n    email\n  }\n}\n"
  }
};
})();

(node as any).hash = "231b7e1f44ac284a6cbf12fb2088b7d6";

export default node;
