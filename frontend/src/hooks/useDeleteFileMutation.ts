import { useMutation } from "react-relay";
import type { PayloadError } from "relay-runtime";
import type { DeleteFileMutation as DeleteFileMutationType } from "../mutations/__generated__/DeleteFileMutation.graphql";
import DeleteFileMutationGraphql from "../mutations/DeleteFileMutation";

interface DeleteFileCallbacks {
  onCompleted?: (
    response: DeleteFileMutationType["response"],
    errors: ReadonlyArray<PayloadError> | null,
  ) => void;
  onError?: (error: Error) => void;
}

export function useDeleteFileMutation() {
  const [commit, isInFlight] = useMutation<DeleteFileMutationType>(
    DeleteFileMutationGraphql,
  );

  const deleteFile = (
    projectId: string,
    path: string,
    callbacks?: DeleteFileCallbacks,
  ) => {
    commit({
      variables: {
        projectId,
        path,
      },
      onCompleted: (response, errors) => {
        callbacks?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        callbacks?.onError?.(err);
      },
    });
  };

  return { deleteFile, isInFlight };
}
