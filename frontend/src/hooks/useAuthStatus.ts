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
  // Relay já faz suspense + error boundary. Não inventa estado.
  const data = useLazyLoadQuery<UseAuthStatusQueryType>(
    query,
    {},
    { fetchPolicy: "network-only" },
  );

  const isAuthenticated = Boolean(data?.me?.id);

  return { isAuthenticated };
}
