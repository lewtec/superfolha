import { useMutation } from 'react-relay';
import type { DeleteFileMutation as DeleteFileMutationType } from '../mutations/__generated__/DeleteFileMutation.graphql';
import DeleteFileMutationGraphql from '../mutations/DeleteFileMutation';

interface UseDeleteFileMutationConfig {
  onCompleted?: (response: DeleteFileMutationType['response'], errors: ReadonlyArray<Error> | null) => void;
  onError?: (error: Error) => void;
}

export function useDeleteFileMutation(config?: UseDeleteFileMutationConfig) {
  const [commit, isInFlight] = useMutation<DeleteFileMutationType>(DeleteFileMutationGraphql);

  const deleteFile = (projectId: string, path: string) => {
    commit({
      variables: {
        projectId,
        path,
      },
      onCompleted: (response, errors) => {
        config?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        config?.onError?.(err);
      },
    });
  };

  return { deleteFile, isInFlight };
}
