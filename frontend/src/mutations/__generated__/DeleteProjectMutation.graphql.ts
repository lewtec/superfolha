/**
 * @generated SignedSource<<f45f872b483770d7c25d582974385e57>>
 * @lightSyntaxTransform
 * @nogrep
 */

/* tslint:disable */
/* eslint-disable */
// @ts-nocheck

import { ConcreteRequest } from "relay-runtime";
export type DeleteProjectMutation$variables = {
  id: string;
};
export type DeleteProjectMutation$data = {
  readonly deleteProject: boolean;
};
export type DeleteProjectMutation = {
  response: DeleteProjectMutation$data;
  variables: DeleteProjectMutation$variables;
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
        kind: "ScalarField",
        name: "deleteProject",
        storageKey: null,
      },
    ];
  return {
    fragment: {
      argumentDefinitions: v0 /*: any*/,
      kind: "Fragment",
      metadata: null,
      name: "DeleteProjectMutation",
      selections: v1 /*: any*/,
      type: "Mutation",
      abstractKey: null,
    },
    kind: "Request",
    operation: {
      argumentDefinitions: v0 /*: any*/,
      kind: "Operation",
      name: "DeleteProjectMutation",
      selections: v1 /*: any*/,
    },
    params: {
      cacheID: "051b860819f177cbf932d7325a5eebe1",
      id: null,
      metadata: {},
      name: "DeleteProjectMutation",
      operationKind: "mutation",
      text: "mutation DeleteProjectMutation(\n  $id: ID!\n) {\n  deleteProject(id: $id)\n}\n",
    },
  };
})();

(node as any).hash = "3c46a5e907815752754dc0767ec09df9";

export default node;
