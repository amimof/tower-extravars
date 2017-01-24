# tower-extravars

This simple little program updates the `extra_vars` field of a `Job Template` in [Ansible Tower](https://www.ansible.com/tower) from an input yaml file. It supports updating, replacing and deleting fields and values. It can update multiple job templates using the same input file. An input file is a `yaml` file created with content beeing extra_vars that is to be applied on an Ansible Tower job template. Continue reading for examples.

## Why?
This is very useful in situations where you want to update the `extra_vars` field of multiple similar Job Templates.

### For example
Imagine that you write a playbook which promotes a docker image from Dev to Qa. The playbook itself is not important here. Using Ansible Tower you define a Job Template. And the extra_vars might look like this:
```yaml
image: myapp:latest
registry: private-registry.company.com
source: dev
target: qa
```

As the number of applications increase in the environment, more and more job templates are added and the `image` field is changed to whatever application which is beeing promoted.

So now you have 20 applications and 20 Job Templates, and now the network admin says that we need to change the hostname of the private-registry.

You can, if you want update all those job templates manually. Or just create the file:
```yaml
image: myapp:latest
registry: super-private-registry.company.com
source: dev
target: qa
```

and run

```
$ tower-extravars -f input.yaml -i 144,145,146,157 -h https://tower.company.com -u admin -p admin 
```

## Update strategies
It is possible to define how the update operation and modifications of your job templates extra_vars occurs. The different strategies are:

### Append
Add missing fields and their values from input file to extra_vars. Below example will add `color: Yellow`.

In Ansible Tower extra_vars:
```yaml
fruit: Banana
description: A yellow fruit
```

In input file
```yaml
fruit: Banana
description: A yellow fruit
color: Yellow
```

### Replace
Replace extra_vars in Ansible Tower with content of the input file. Below example will replace the entire extra_vars field with the input file.

In Ansible Tower extra_vars:
```yaml
fruit: Banana
description: A yellow fruit
color: Yellow
```

In input file
```yaml
fruit: Apple
description: This is not a Banana
color: Red
```

### Update
Replaces extra_vars *values* in Ansible Tower with content of the input file, if the field exists in both places. Below replaces the value of the field `color`

In Ansible Tower extra_vars:
```yaml
fruit: Apple
description: This is not a Banana
color: Red
```

In input file
```yaml
fruit: Apple
description: This is not a Banana
color: Green
```

### Delete
Delete all fields that are defined in the input file from extra_vars in Ansible Tower. Below will remove `description`. This strategy will ignore the values and only compare any mathing field names. This is the default strategy.

In Ansible Tower extra_vars:
```yaml
fruit: Apple
description: This is not a Banana
color: Green
```

In input file
```yaml
description: This is not a Banana
```



## Installation

### Binary
Use the compiled binaries under [Releases](https://github.com/amimof/tower-extravars/releases) for your platform.

### From source
Requires [Go](https://golang.org/)
```
$ go get -t github.com/amimof/tower-extravars
$ go build github.com/amimof/tower-extravars
```

## Usage

```
```