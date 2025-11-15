import { useMutation } from 'react-relay';
import type { SaveFileMutation as SaveFileMutationType } from '../mutations/__generated__/SaveFileMutation.graphql';
import SaveFileMutationGraphql from '../mutations/SaveFileMutation';

interface UseSaveFileMutationConfig {
  onCompleted?: (response: SaveFileMutationType['response'], errors: ReadonlyArray<Error> | null) => void;
  onError?: (error: Error) => void;
}

export function useSaveFileMutation(config?: UseSaveFileMutationConfig) {
  const [commit, isInFlight] = useMutation<SaveFileMutationType>(SaveFileMutationGraphql);

  const saveFile = (projectId: string, path: string, content: string) => {
    commit({
      variables: {
        projectId,
        path,
        content,
      },
      onCompleted: (response, errors) => {
        config?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        config?.onError?.(err);
      },
    });
  };

  return { saveFile, isInFlight };
}
