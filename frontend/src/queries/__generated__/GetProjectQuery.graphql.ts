/**
 * @generated SignedSource<<f65653da192094fee72ce84fdcd3878d>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from "relay-runtime";
export type GetProjectQuery$variables = {
  id: string;
};
export type GetProjectQuery$data = {
  readonly project:
    | {
        readonly files: ReadonlyArray<{
          readonly content: string | null | undefined;
          readonly isBinary: boolean;
          readonly isTooBig: boolean;
          readonly path: string;
          readonly size: number;
        }>;
        readonly history: ReadonlyArray<{
          readonly author: string;
          readonly date: any;
          readonly hash: string;
          readonly message: string;
        }>;
        readonly id: string;
        readonly name: string;
      }
    | null
    | undefined;
};
export type GetProjectQuery = {
  response: GetProjectQuery$data;
  variables: GetProjectQuery$variables;
};

const node: ConcreteRequest = (function () {
  var v0 = [
      {
        defaultValue: null,
        kind: "LocalArgument",
        name: "id",
      },
    ],
    v1 = [
      {
        alias: null,
        args: [
          {
            kind: "Variable",
            name: "id",
            variableName: "id",
          },
        ],
        concreteType: "Project",
        kind: "LinkedField",
        name: "project",
        plural: false,
        selections: [
          {
            alias: null,
            args: null,
            kind: "ScalarField",
            name: "id",
            storageKey: null,
          },
          {
            alias: null,
            args: null,
            kind: "ScalarField",
            name: "name",
            storageKey: null,
          },
          {
            alias: null,
            args: null,
            concreteType: "File",
            kind: "LinkedField",
            name: "files",
            plural: true,
            selections: [
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "path",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "content",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "size",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "isTooBig",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "isBinary",
                storageKey: null,
              },
            ],
            storageKey: null,
          },
          {
            alias: null,
            args: null,
            concreteType: "Commit",
            kind: "LinkedField",
            name: "history",
            plural: true,
            selections: [
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "hash",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "message",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "author",
                storageKey: null,
              },
              {
                alias: null,
                args: null,
                kind: "ScalarField",
                name: "date",
                storageKey: null,
              },
            ],
            storageKey: null,
          },
        ],
        storageKey: null,
      },
    ];
  return {
    fragment: {
      argumentDefinitions: v0 /*: any*/,
      kind: "Fragment",
      metadata: null,
      name: "GetProjectQuery",
      selections: v1 /*: any*/,
      type: "Query",
      abstractKey: null,
    },
    kind: "Request",
    operation: {
      argumentDefinitions: v0 /*: any*/,
      kind: "Operation",
      name: "GetProjectQuery",
      selections: v1 /*: any*/,
    },
    params: {
      cacheID: "f1b732c46f145bb54a3430160d3cea65",
      id: null,
      metadata: {},
      name: "GetProjectQuery",
      operationKind: "query",
      text: "query GetProjectQuery(\n  $id: ID!\n) {\n  project(id: $id) {\n    id\n    name\n    files {\n      path\n      content\n      size\n      isTooBig\n      isBinary\n    }\n    history {\n      hash\n      message\n      author\n      date\n    }\n  }\n}\n",
    },
  };
})();

(node as any).hash = "e90e6a5f588d7d96099ab7778d75686d";

export default node;
