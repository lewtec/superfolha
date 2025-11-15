import { graphql } from 'react-relay';

export default graphql`
  query GetProjectQuery($id: ID!) {
    project(id: $id) {
      id
      name
    }
  }
`;
