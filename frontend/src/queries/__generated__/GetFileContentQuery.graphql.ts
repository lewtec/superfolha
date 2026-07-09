/**
 * @generated SignedSource<<3cd6a4825d9661fc57328aa9f5d2454b>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from "relay-runtime";
export type GetFileContentQuery$variables = {
  id: string;
  path: string;
};
export type GetFileContentQuery$data = {
  readonly project:
    | {
        readonly file:
          | {
              readonly content: string | null | undefined;
            }
          | null
          | undefined;
      }
    | null
    | undefined;
};
export type GetFileContentQuery = {
  response: GetFileContentQuery$data;
  variables: GetFileContentQuery$variables;
};

const node: ConcreteRequest = (function () {
  var v0 = [
      {
        defaultValue: null,
        kind: "LocalArgument",
        name: "id",
      },
      {
        defaultValue: null,
        kind: "LocalArgument",
        name: "path",
      },
    ],
    v1 = [
      {
        kind: "Variable",
        name: "id",
        variableName: "id",
      },
    ],
    v2 = {
      alias: null,
      args: [
        {
          kind: "Variable",
          name: "path",
          variableName: "path",
        },
      ],
      concreteType: "File",
      kind: "LinkedField",
      name: "file",
      plural: false,
      selections: [
        {
          alias: null,
          args: null,
          kind: "ScalarField",
          name: "content",
          storageKey: null,
        },
      ],
      storageKey: null,
    };
  return {
    fragment: {
      argumentDefinitions: v0 /*: any*/,
      kind: "Fragment",
      metadata: null,
      name: "GetFileContentQuery",
      selections: [
        {
          alias: null,
          args: v1 /*: any*/,
          concreteType: "Project",
          kind: "LinkedField",
          name: "project",
          plural: false,
          selections: [v2 /*: any*/],
          storageKey: null,
        },
      ],
      type: "Query",
      abstractKey: null,
    },
    kind: "Request",
    operation: {
      argumentDefinitions: v0 /*: any*/,
      kind: "Operation",
      name: "GetFileContentQuery",
      selections: [
        {
          alias: null,
          args: v1 /*: any*/,
          concreteType: "Project",
          kind: "LinkedField",
          name: "project",
          plural: false,
          selections: [
            v2 /*: any*/,
            {
              alias: null,
              args: null,
              kind: "ScalarField",
              name: "id",
              storageKey: null,
            },
          ],
          storageKey: null,
        },
      ],
    },
    params: {
      cacheID: "6b3f36ccf10197c54a2a594b19d67eef",
      id: null,
      metadata: {},
      name: "GetFileContentQuery",
      operationKind: "query",
      text: "query GetFileContentQuery(\n  $id: ID!\n  $path: String!\n) {\n  project(id: $id) {\n    file(path: $path) {\n      content\n    }\n    id\n  }\n}\n",
    },
  };
})();

(node as any).hash = "d7c6ee63ca5f3305c4833704b7b26651";

export default node;
