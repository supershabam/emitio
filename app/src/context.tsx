import * as React from "react";
import { Action } from "./actions";
import { State } from "./state";
import { Observable, Subscription } from "rxjs";

export const { Provider, Consumer } = React.createContext({});

export const connect = (
  WrappedComponent: any,
  mapState$ToProps?: any,
  mapDispatchToProps?: any
) => {
  interface ConnectProps {
    store: {
      dispatch: (a: Action) => null;
      state$: Observable<State>;
    };
  }
  class Connect extends React.Component<ConnectProps> {
    static WrappedComponent = WrappedComponent;
    subs: Array<Subscription>;
    constructor(props: ConnectProps) {
      super(props);
      this.state = {};
      this.subs = [];
    }
    componentDidMount() {
      for (let key in mapState$ToProps) {
        this.subs = this.subs.concat([
          mapState$ToProps[key](this.props.store.state$).subscribe(value => {
            this.setState(prevState => {
              return { ...prevState, [key]: value };
            });
          })
        ]);
      }
    }
    componentWillUnmount() {
      for (let sub of this.subs) {
        sub.unsubscribe();
      }
    }
    render() {
      return (
        <WrappedComponent
          state$={this.props.store.state$}
          dispatch={this.props.store.dispatch}
          {...(this.props, this.state)}
        />
      );
    }
  }
  return props => {
    return <Consumer>{store => <Connect store={store} {...props} />}</Consumer>;
  };
};
