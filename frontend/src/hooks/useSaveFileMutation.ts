import { useMutation } from "react-relay";
import type { PayloadError } from "relay-runtime";
import type { SaveFileMutation as SaveFileMutationType } from "../mutations/__generated__/SaveFileMutation.graphql";
import SaveFileMutationGraphql from "../mutations/SaveFileMutation";

interface SaveFileCallbacks {
  onCompleted?: (
    response: SaveFileMutationType["response"],
    errors: ReadonlyArray<PayloadError> | null,
  ) => void;
  onError?: (error: Error) => void;
}

export function useSaveFileMutation() {
  // Removed config from here
  const [commit, isInFlight] = useMutation<SaveFileMutationType>(
    SaveFileMutationGraphql,
  );

  const saveFile = (
    projectId: string,
    path: string,
    content: string,
    callbacks?: SaveFileCallbacks,
  ) => {
    // Added callbacks parameter
    commit({
      variables: {
        projectId,
        path,
        content,
      },
      onCompleted: (response, errors) => {
        callbacks?.onCompleted?.(response, errors); // Use callbacks parameter
      },
      onError: (err) => {
        callbacks?.onError?.(err); // Use callbacks parameter
      },
    });
  };

  return { saveFile, isInFlight };
}
