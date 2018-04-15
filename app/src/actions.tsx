interface LoginRequest {
  kind: "LoginRequest";
  username: string;
  password: string;
}

interface LoginSuccess {
  kind: "LoginSuccess";
  id: string;
  name: string;
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

interface RefreshServicesRequest {
  kind: "RefreshServicesRequest";
  userID: string;
}

interface RefreshServicesSuccess {
  kind: "RefreshServicesSuccess";
  services?: string[];
}

interface SelectService {
  kind: "SelectService";
  selected: string;
}

export type Action =
  | LoginRequest
  | LoginSuccess
  | LoginError
  | Logout
  | RefreshServicesRequest
  | RefreshServicesSuccess
  | SelectService;
