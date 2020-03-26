select a.address,
       eis.epoch,
       dis.name                                  state,
       coalesce(prevdis.name, '')                prev_state,
       coalesce(ei.approved, false),
       coalesce(ei.missed, false),
       coalesce(ei.short_point, 0),
       coalesce(ei.short_flips, 0),
       coalesce(ei.total_short_point, 0),
       coalesce(ei.total_short_flips, 0),
       coalesce(ei.long_point, 0),
       coalesce(ei.long_flips, 0),
       coalesce(ei.required_flips, 0)            required_flips,
       coalesce(ei.made_flips, 0)                made_flips,
       coalesce(ei.available_flips, 0)           available_flips,
       coalesce((select sum(vr.balance + vr.stake)
                 from validation_rewards vr
                 where vr.ei_address_state_id = eis.address_state_id
                   and ei.epoch = eis.epoch), 0) total_validation_reward
from epoch_identity_states eis
         join addresses a on a.id = eis.address_id
         left join address_states prevs on prevs.id = eis.prev_id
         left join epoch_identities ei on ei.epoch = eis.epoch and ei.address_state_id = eis.address_state_id
         join dic_identity_states dis on dis.id = eis.state
         left join dic_identity_states prevdis on prevdis.id = prevs.state
where lower(a.address) = lower($1)
order by eis.epoch desc
limit $3
offset
$2