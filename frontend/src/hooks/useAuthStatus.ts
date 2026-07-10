import { graphql, useLazyLoadQuery } from "react-relay";
import type { useAuthStatusQuery as UseAuthStatusQueryType } from "./__generated__/useAuthStatusQuery.graphql";

const query = graphql`
  query useAuthStatusQuery {
    me {
      id
      email
    }
  }
`;

export function useAuthStatus() {
  const data = useLazyLoadQuery<UseAuthStatusQueryType>(
    query,
    {},
    { fetchPolicy: "store-or-network" },
  );

  const me = data?.me ?? null;
  const isAuthenticated = Boolean(me?.id);

  return {
    isAuthenticated,
    userId: me?.id ?? null,
    email: me?.email ?? null,
  };
}
