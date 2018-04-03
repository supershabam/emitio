import * as React from "react";

export const APIContext = React.createContext();

export const withAPI = WrappedComponent => props => (
  <APIContext.Consumer>
    {api => <WrappedComponent {...api} {...props} />}
  </APIContext.Consumer>
);
