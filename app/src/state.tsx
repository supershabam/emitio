export interface State {
  user?: {
    id: string;
    name: string;
  };
  service: {
    refreshing: boolean;
    services: Service[];
    selected?: {
      service: Service;
      refreshingColumns: boolean;
      columns: Column[];
      calculate?: Column;
    };
  };
}

export interface Service {
  id: string;
  name: string;
}

export interface NumericColumn {
  kind: "NumericColumn";
  field: string;
}

export type Column = NumericColumn;
