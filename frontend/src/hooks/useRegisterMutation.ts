import { useMutation } from "react-relay";
import type { PayloadError } from "relay-runtime";
import type { RegisterMutation as RegisterMutationType } from "../mutations/__generated__/RegisterMutation.graphql";
import RegisterMutationGraphql from "../mutations/RegisterMutation";

interface UseRegisterMutationConfig {
  onCompleted?: (
    response: RegisterMutationType["response"],
    errors: ReadonlyArray<PayloadError> | null,
  ) => void;
  onError?: (error: Error) => void;
}

export function useRegisterMutation(config?: UseRegisterMutationConfig) {
  const [commit, isInFlight] = useMutation<RegisterMutationType>(
    RegisterMutationGraphql,
  );

  const register = (email: string, password: string) => {
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

  return { register, isInFlight };
}
