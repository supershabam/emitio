import { Service, Column } from "./state";

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

interface RefreshColumnsRequest {
  kind: "RefreshColumnsRequest";
  userID: string;
  serviceID: string;
}

interface RefreshColumnsSuccess {
  kind: "RefreshColumnsSuccess";
  columns: Column[];
}

interface SelectCalculateColumn {
  kind: "SelectCalculateColumn";
  column: Column;
}

interface RefreshServicesRequest {
  kind: "RefreshServicesRequest";
  userID: string;
}

interface RefreshServicesSuccess {
  kind: "RefreshServicesSuccess";
  services?: Service[];
}

interface SelectService {
  kind: "SelectService";
  service: Service;
}

export type Action =
  | LoginRequest
  | LoginSuccess
  | LoginError
  | Logout
  | RefreshServicesRequest
  | RefreshServicesSuccess
  | SelectService
  | RefreshColumnsRequest
  | RefreshColumnsSuccess
  | SelectCalculateColumn;
