# 0-OS, a bit of history and introduction to Version 2

## Once upon a time
----
A few years ago, we were trying to come up with some solutions to the problem of self-healing IT.  
We boldly started that : the current model of cloud computing in huge data-centers is not going to be able to scale to fit the demand in IT capacity.

The approach we took to solve this problem was to enable localized compute and storage units at the edge of the network, close to where it is needed.  
That basically meant that if we were to deploy physical hardware to the edges, nearby the users, we would have to allow information providers to deploy their solutions on that edge network and hardware. That means also sharing hardware resources between users, where we would have to make damn sure noone can peek around in things that are not his.

When we talk about sharing capacity in a secure environment, virtualization comes to mind. It's not a new technology and it has been around for quite some time. This solution comes with a cost though. Virtual machines, emulating a full hardware platform on real hardware is costly in terms of used resources, and eat away at the already scarce resources we want to provide for our users.

Containerizing technologies were starting to get some hype at the time. Containers provide for basically the same level of isolation as Full Virtualisation, but are a lot less expensive in terms of resource utilization.

With that in mind, we started designing the first version of 0-OS. The required features were:

- be able to be fully in control of the hardware
- give the possibility to different users to share the same hardware
- deploy this capacity at the edge, close to where it is needed
- the System needs to self-heal. Because of their location and sheer scale, manual maintenance was not an option. Self-healing is a broad topic, and will require a lot of experience and fine-tuning, but it was meant to culminate at some point so that most of the actions that sysadmins execute, would be automated.
- Have an a small as possible attack surface, as well for remote types of attack, as well as protecting users from each-other

The result of that thought process resulted in 0-OS v1. A linux kernel with the minimal components on top that allows to provide for these features.

In the first incantation of 0-OS, the core framework was a single big binary that got started as the first process of the system (PID 1). All the managment features were exposed through an API that was only accessible locally.

The idea was to have an orchestration system running on top that was going to be responsible to deploy Virtual Machines and Containers on the system using that API.

This API exposes 3 main primitives:

- networking: zerotier, vlan, macvlan, bridge, openvswitch...
- storage: plain disk, 0-db, ...
- compute: VM, containers

That was all great and it allowed us to learn a lot. But some limitations started to appear. Here is a non exhaustive list of the limitations we had to face after a couple of years of utilization:

- Difficulty to push new versions and fixes on the nodes. The fact that 0-OS was a single process running as PID 1, forced us to completely reboot the node every time we wanted to push an update.
- The API, while powerful, still required to have some logic on top to actually deploy usable solutions.
- We noticed that some features we implemented were never or extremely rarely used. This was just increasing the possible attack surface for no real benefits.
- The main networking solution we choose at the time, zerotier, was not scaling as well as we hoped for.
- We wrote a lot of code ourselves, instead of relying on already existing open source libraries that would have made that task a lot easier, but also, these libraries were a lot more mature and have had a lot more exposure for ironing out possible bugs and vulnerabilities than we could have created and tested ourselves with the little resources we have at hand.

## Now what ?
With the knowledge and lessons gathered during these first years of usage, we
concluded that trying to fix the already existing codebase would be cumbersome
and we also wanted to avoid any technical debt that could haunt us for years
after. So we decided for a complete rewrite of that stack, taking a new and
fully modular approach, where every component could be easily replaced and
upgraded without the need for a reboot.  

Hence Version 2 saw the light of day.  

Instead of trial and error, and muddling along trying to fit new features in
that big monolithic codebase, we wanted to be sure that the components were
reduced to a more manageable size, having a clearly cut Domain Separation.

Instead of creating solutions waiting for a problem, we started looking at things the other way around. Which is logical, as by now, we learned what the real puzzles to solve were, albeit sometimes by painful experience. 

## Tadaa! 
----
The [first commit](https://github.com/threefoldtech/zosv2/commit/7b783c888673d1e9bc400e4abbb17272e995f5a4) of the v2 repository took place the 11 of February 2019.
We are now 6 months in, and about to bake the first release of 0-OS v2.  
Clocking in at almost 27KLoc, it was a very busy half-year. (admitted, there are the spec and docs too in that count ;-) )

Let's go over the main design decisions that were made and explain briefly each component.  

While this is just an introduction, we'll add more articles digging deeper in the technicalities and approaches of each component.  

## Solutions to puzzles (there are no problems)
----
**UPDATES** 

One of the first puzzles we wanted to solve was the difficulty to push upgrades.  
In order to solve that, we designed 0-OS components as completely stand-alone modules. Each subsystem, be it storage, networking, containers/VMs, is managed by it's own component (mostly a daemon), and communicate with each-other through a local bus. And as we said, each component can then be upgraded separately, together with the necessary data migrations that could be required.

**WHAT API?** 

The second big change is our approach to the API, or better, lack thereof.  
In V2 we dropped the idea to expose the primitives of the Node over an API.  
Instead, all the required knowledge to deploy workloads is directly embedded in 0-OS.  
So in order to have the node deploy a workload, we have created a blueprint like system where the user describes what his requirements in terms of compute power, storage and networking are, and the node applies that blueprint to make it reality.  
That approach has a few advantages:
  - It greatly reduces the attack surface of the node because there is no more direct interaction between a user and a node.
  - And it also allows us to have a greater control over how things are organized in the node itself. The node being its own boss, can decide to re-organize itself whenever needed to optimize the capacity it can provide.
  - Having a blueprint with requirements, gives the grid the possibility to verify that blueprint on multiple levels before applying it. That is: as well on top level as on node level a blueprint can be verified for validity and signatures before any other action will be executed.

**PING**  

The last major change is how we want to handle networking.  
The solution used during the lifetime of V1 exposed its limitations when we started scaling our networks to hundreds of nodes.  
So here again we started from scratch and created our own overlay network solution.  
That solution is based on the 'new kid on the block' in terms of VPN: [Wireguard](https://wireguard.io) and it's approach and usage will be fully explained in the next 0-OS article.  
For the eager ones of you, there are some specifications and also some documentation [here](https://github.com/threefoldtech/zosv2/tree/master/docs/network) and [there](https://github.com/threefoldtech/zosv2/tree/master/specs/network).

## That's All, Folks (for now)
So this little article as an intro to the brave new world of 0-OS.  
The Zero-OS team engages itself to regularly keep you updated on it's progress, the new features that will surely be added, and for the so inclined, add a lot more content for techies on how to actually use that novel beast.

[Till next time](https://youtu.be/b9434BoGkNQ)
