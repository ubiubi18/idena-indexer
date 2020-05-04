select a.address,
       ei.epoch,
       dis.name                   state,
       coalesce(prevdis.name, '') prev_state,
       ei.approved,
       ei.missed,
       ei.short_point,
       ei.short_flips,
       ei.total_short_point,
       ei.total_short_flips,
       ei.long_point,
       ei.long_flips,
       ei.required_flips,
       ei.made_flips,
       ei.available_flips,
       ei.total_validation_reward,
       ei.birth_epoch
from epoch_identities ei
         join address_states s on s.id = ei.address_state_id
         join addresses a on a.id = s.address_id and lower(a.address) = lower($1)
         join dic_identity_states dis on dis.id = s.state
         left join address_states prevs on prevs.id = s.prev_id
         left join dic_identity_states prevdis on prevdis.id = prevs.state
order by ei.address_state_id desc
limit $3 offset $2