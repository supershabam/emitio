export interface State {
  user?: {
    id: string;
    name: string;
  };
  service: {
    refreshing: boolean;
    selected?: string;
    services: string[];
  };
}
