import { useMutation } from "react-relay";
import type { PayloadError } from "relay-runtime";
import type { LoginMutation as LoginMutationType } from "../mutations/__generated__/LoginMutation.graphql";
import LoginMutationGraphql from "../mutations/LoginMutation";

interface UseLoginMutationConfig {
  onCompleted?: (
    response: LoginMutationType["response"],
    errors: ReadonlyArray<PayloadError> | null,
  ) => void;
  onError?: (error: Error) => void;
}

export function useLoginMutation(config?: UseLoginMutationConfig) {
  const [commit, isInFlight] =
    useMutation<LoginMutationType>(LoginMutationGraphql);

  const login = (email: string, password: string) => {
    commit({
      variables: {
        email,
        password,
      },
      onCompleted: (response, errors) => {
        config?.onCompleted?.(response, errors);
      },
      onError: (err) => {
        config?.onError?.(err);
      },
    });
  };

  return { login, isInFlight };
}
