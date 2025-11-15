import { graphql } from 'react-relay';

export default graphql`
  query GetFilesQuery($projectId: ID!) {
    files(projectId: $projectId) {
      path
      content
    }
  }
`;
