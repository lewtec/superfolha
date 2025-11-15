import { useLazyLoadQuery } from 'react-relay';
import type { GetFilesQuery as GetFilesQueryType } from '../queries/__generated__/GetFilesQuery.graphql';
import GetFilesQueryGraphql from '../queries/GetFilesQuery';

interface UseGetFilesQueryConfig {
  projectId: string;
  fetchPolicy?: 'store-and-network' | 'network-only' | 'store-only' | 'store-or-network';
}

export function useGetFilesQuery(config: UseGetFilesQueryConfig) {
  const data = useLazyLoadQuery<GetFilesQueryType>(
    GetFilesQueryGraphql,
    { projectId: config.projectId },
    { fetchPolicy: config.fetchPolicy || 'store-and-network' }
  );

  return data;
}
