/**
 * @generated SignedSource<<7a3d0b747b3256dcabdb1abec6615fe7>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from "relay-runtime";
export type SaveFileMutation$variables = {
  content: string;
  path: string;
  projectId: string;
};
export type SaveFileMutation$data = {
  readonly saveFile: {
    readonly content: string | null | undefined;
    readonly path: string;
  };
};
export type SaveFileMutation = {
  response: SaveFileMutation$data;
  variables: SaveFileMutation$variables;
};

const node: ConcreteRequest = (function () {
  var v0 = {
      defaultValue: null,
      kind: "LocalArgument",
      name: "content",
    },
    v1 = {
      defaultValue: null,
      kind: "LocalArgument",
      name: "path",
    },
    v2 = {
      defaultValue: null,
      kind: "LocalArgument",
      name: "projectId",
    },
    v3 = [
      {
        alias: null,
        args: [
          {
            kind: "Variable",
            name: "content",
            variableName: "content",
          },
          {
            kind: "Variable",
            name: "path",
            variableName: "path",
          },
          {
            kind: "Variable",
            name: "projectId",
            variableName: "projectId",
          },
        ],
        concreteType: "File",
        kind: "LinkedField",
        name: "saveFile",
        plural: false,
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
        ],
        storageKey: null,
      },
    ];
  return {
    fragment: {
      argumentDefinitions: [v0 /*: any*/, v1 /*: any*/, v2 /*: any*/],
      kind: "Fragment",
      metadata: null,
      name: "SaveFileMutation",
      selections: v3 /*: any*/,
      type: "Mutation",
      abstractKey: null,
    },
    kind: "Request",
    operation: {
      argumentDefinitions: [v2 /*: any*/, v1 /*: any*/, v0 /*: any*/],
      kind: "Operation",
      name: "SaveFileMutation",
      selections: v3 /*: any*/,
    },
    params: {
      cacheID: "a47ad4e655b21c00d7dcfaf8a66b2e6d",
      id: null,
      metadata: {},
      name: "SaveFileMutation",
      operationKind: "mutation",
      text: "mutation SaveFileMutation(\n  $projectId: ID!\n  $path: String!\n  $content: String!\n) {\n  saveFile(projectId: $projectId, path: $path, content: $content) {\n    path\n    content\n  }\n}\n",
    },
  };
})();

(node as any).hash = "e0b34f51b41e734841d4701bc00cbc66";

export default node;
