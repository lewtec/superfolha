/**
 * @generated SignedSource<<d46482eeb4d13b88747e077407ae69bc>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from 'relay-runtime';
export type ProjectsQuery$variables = Record<PropertyKey, never>;
export type ProjectsQuery$data = {
  readonly projects: ReadonlyArray<{
    readonly createdAt: any;
    readonly id: string;
    readonly name: string;
    readonly updatedAt: any;
  }>;
};
export type ProjectsQuery = {
  response: ProjectsQuery$data;
  variables: ProjectsQuery$variables;
};

const node: ConcreteRequest = (function(){
var v0 = [
  {
    "alias": null,
    "args": null,
    "concreteType": "Project",
    "kind": "LinkedField",
    "name": "projects",
    "plural": true,
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
    "argumentDefinitions": [],
    "kind": "Fragment",
    "metadata": null,
    "name": "ProjectsQuery",
    "selections": (v0/*: any*/),
    "type": "Query",
    "abstractKey": null
  },
  "kind": "Request",
  "operation": {
    "argumentDefinitions": [],
    "kind": "Operation",
    "name": "ProjectsQuery",
    "selections": (v0/*: any*/)
  },
  "params": {
    "cacheID": "5dc7f6e1d0054f704a34defba0349b96",
    "id": null,
    "metadata": {},
    "name": "ProjectsQuery",
    "operationKind": "query",
    "text": "query ProjectsQuery {\n  projects {\n    id\n    name\n    createdAt\n    updatedAt\n  }\n}\n"
  }
};
})();

(node as any).hash = "4f00ec861e7b7bf026977974724c2bd7";

export default node;
