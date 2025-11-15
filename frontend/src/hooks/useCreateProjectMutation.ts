import { useMutation } from 'react-relay';
import type { CreateProjectMutation as CreateProjectMutationType } from '../mutations/__generated__/CreateProjectMutation.graphql';
import CreateProjectMutationGraphql from '../mutations/CreateProjectMutation';

interface UseCreateProjectMutationConfig {
  onCompleted?: (response: CreateProjectMutationType['response'], errors: ReadonlyArray<Error> | null) => void;
  onError?: (error: Error) => void;
}

export function useCreateProjectMutation(config?: UseCreateProjectMutationConfig) {
  const [commit, isInFlight] = useMutation<CreateProjectMutationType>(CreateProjectMutationGraphql);

  const createProject = (name: string) => {
    commit({
      variables: {
        name,
      },
      onCompleted: (response, errors) => {
        config?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        config?.onError?.(err);
      },
    });
  };

  return { createProject, isInFlight };
}
