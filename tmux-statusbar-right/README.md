# tmux status right

## Purpose

Prints the selected k8s cluster and openstack env.

If multiple kubeconfig files are added to the `KUBECONFIG` env var
(delimited by `:`) it prints `KUBECONFIG+!` to inform the user.

In case the variable is empty but the default file exists it informs the
user by stating `KUBECONFIG= `.

## Where it is used

Executable used [here](https://github.com/diepfote/dot-files/blob/aed558943e888cc6b32eacdb9f64ca687f358869/.tmux.conf#L44).
