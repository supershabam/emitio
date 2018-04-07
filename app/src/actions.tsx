interface LoginRequest {
  kind: "LoginRequest";
  username: string;
  password: string;
}

interface LoginSuccess {
  kind: "LoginSuccess";
  username: string;
}

interface LoginError {
  kind: "LoginError";
  error: string;
}

interface Logout {
  kind: "Logout";
}

interface State {
  drawer: {
    open: boolean;
  };
  username?: string;
}

export type Action = LoginRequest | LoginSuccess | LoginError | Logout;
