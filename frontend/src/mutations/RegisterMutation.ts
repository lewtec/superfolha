import { graphql } from 'react-relay';

export default graphql`
  mutation RegisterMutation($email: String!, $password: String!) {
    register(email: $email, password: $password) {
      token
      user {
        id
        email
      }
    }
  }
`;
