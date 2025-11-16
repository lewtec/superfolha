import { graphql } from 'react-relay';

export default graphql`
  query GetProjectQuery($id: ID!) {
    project(id: $id) {
      id
      name
      files {
        path
        content
        size
        isTooBig
        isBinary
      }
      history {
        hash
        message
        author
        date
      }
    }
  }
`;
