import { graphql } from 'react-relay';

export default graphql`
  mutation DeleteFileMutation($projectId: ID!, $path: String!) {
    deleteFile(projectId: $projectId, path: $path)
  }
`;
