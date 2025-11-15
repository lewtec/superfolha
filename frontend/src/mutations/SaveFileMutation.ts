import { graphql } from 'react-relay';

export default graphql`
  mutation SaveFileMutation($projectId: ID!, $path: String!, $content: String!) {
    saveFile(projectId: $projectId, path: $path, content: $content) {
      path
      content
    }
  }
`;
