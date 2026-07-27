import { useMutation } from "react-relay";
import type { PayloadError, SelectorStoreUpdater } from "relay-runtime";
import type { DeleteProjectMutation as DeleteProjectMutationType } from "../mutations/__generated__/DeleteProjectMutation.graphql";
import DeleteProjectMutationGraphql from "../mutations/DeleteProjectMutation";

interface DeleteProjectCallbacks {
  onCompleted?: (
    response: DeleteProjectMutationType["response"],
    errors: ReadonlyArray<PayloadError> | null,
  ) => void;
  onError?: (error: Error) => void;
  updater?: SelectorStoreUpdater<DeleteProjectMutationType["response"]>;
}

export function useDeleteProjectMutation() {
  const [commit, isInFlight] = useMutation<DeleteProjectMutationType>(
    DeleteProjectMutationGraphql,
  );

  const deleteProject = (id: string, callbacks?: DeleteProjectCallbacks) => {
    commit({
      variables: {
        id,
      },
      onCompleted: (response, errors) => {
        callbacks?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        callbacks?.onError?.(err);
      },
      updater: callbacks?.updater,
    });
  };

  return { deleteProject, isInFlight };
}
