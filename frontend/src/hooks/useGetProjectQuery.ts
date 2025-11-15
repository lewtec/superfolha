import { useLazyLoadQuery } from 'react-relay';
import type { GetProjectQuery as GetProjectQueryType } from '../queries/__generated__/GetProjectQuery.graphql';
import GetProjectQueryGraphql from '../queries/GetProjectQuery';

interface UseGetProjectQueryConfig {
  id: string;
  fetchPolicy?: 'store-and-network' | 'network-only' | 'store-only' | 'store-or-network';
}

export function useGetProjectQuery(config: UseGetProjectQueryConfig) {
  const data = useLazyLoadQuery<GetProjectQueryType>(
    GetProjectQueryGraphql,
    { id: config.id },
    { fetchPolicy: config.fetchPolicy || 'store-and-network' }
  );

  return data;
}
