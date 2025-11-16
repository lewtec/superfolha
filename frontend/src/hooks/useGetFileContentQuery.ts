import { fetchQuery, useRelayEnvironment } from 'react-relay';
import type { GetFileContentQuery as GetFileContentQueryType } from '../queries/__generated__/GetFileContentQuery.graphql';
import GetFileContentQueryGraphql from '../queries/GetFileContentQuery';
import { useCallback } from 'react';

export function useGetFileContent() {
  const environment = useRelayEnvironment();

  const getFileContent = useCallback(
    (variables: { id: string; path: string }) => {
      return fetchQuery<GetFileContentQueryType>(
        environment,
        GetFileContentQueryGraphql,
        variables
      ).toPromise();
    },
    [environment]
  );

  return { getFileContent };
}
