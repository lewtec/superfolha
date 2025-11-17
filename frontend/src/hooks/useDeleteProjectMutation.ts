import { useMutation } from "react-relay";
import type { DeleteProjectMutation as DeleteProjectMutationType } from "../mutations/__generated__/DeleteProjectMutation.graphql";
import DeleteProjectMutationGraphql from "../mutations/DeleteProjectMutation";

interface UseDeleteProjectMutationConfig {
  onCompleted?: (
    response: DeleteProjectMutationType["response"],
    errors: ReadonlyArray<Error> | null,
  ) => void;
  onError?: (error: Error) => void;
  updater?: DeleteProjectMutationType["updater"];
}

export function useDeleteProjectMutation(
  config?: UseDeleteProjectMutationConfig,
) {
  const [commit, isInFlight] = useMutation<DeleteProjectMutationType>(
    DeleteProjectMutationGraphql,
  );

  const deleteProject = (id: string) => {
    commit({
      variables: {
        id,
      },
      onCompleted: (response, errors) => {
        config?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        config?.onError?.(err);
      },
      updater: config?.updater,
    });
  };

  return { deleteProject, isInFlight };
}
