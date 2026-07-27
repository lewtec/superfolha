import { useMutation } from "react-relay";
import type { PayloadError } from "relay-runtime";
import type { CommitProjectMutation as CommitProjectMutationType } from "../mutations/__generated__/CommitProjectMutation.graphql";
import CommitProjectMutationGraphql from "../mutations/CommitProjectMutation";

interface CommitProjectCallbacks {
  onCompleted?: (
    response: CommitProjectMutationType["response"],
    errors: ReadonlyArray<PayloadError> | null,
  ) => void;
  onError?: (error: Error) => void;
}

export function useCommitProjectMutation() {
  const [commit, isInFlight] = useMutation<CommitProjectMutationType>(
    CommitProjectMutationGraphql,
  );

  const commitProject = (
    projectId: string,
    message: string,
    callbacks?: CommitProjectCallbacks,
  ) => {
    commit({
      variables: {
        projectId,
        message,
      },
      onCompleted: (response, errors) => {
        callbacks?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        callbacks?.onError?.(err);
      },
    });
  };

  return { commitProject, isInFlight };
}
