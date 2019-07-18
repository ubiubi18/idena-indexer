select a.address,
       s.state,
       ei.approved,
       ei.missed,
       (case
            when ei.short_flips != 0 then ei.short_point / ei.short_flips
            else 0
           end) respScore,
       (case
            when COALESCE(allf.fCount, 0) != 0 then (COALESCE(qf.fCount, 0) + 0.5 * COALESCE(wqf.fCount, 0)) /
                                                    allf.fCount
            else 0
           end) authorScore

from epoch_identities ei
         join epochs e on e.id = ei.epoch_id
         join address_states s on s.id = ei.address_state_id
         join addresses a on a.id = s.address_id

         left join (select e.id e_id, a.id a_id, count(*) fCount
                    from flips f
                             join transactions t on t.id = f.tx_id
                             join blocks b on b.id = t.block_id
                             join epochs e on e.id = b.epoch_id
                             join addresses a on a.id = t.from
                    where f.status = 'Qualified'
                    group by a.id, e.id) qf on qf.a_id = a.id and qf.e_id = e.id

         left join (select e.id e_id, a.id a_id, count(*) fCount
                    from flips f
                             join transactions t on t.id = f.tx_id
                             join blocks b on b.id = t.block_id
                             join epochs e on e.id = b.epoch_id
                             join addresses a on a.id = t.from
                    where f.status = 'WeaklyQualified'
                    group by a.id, e.id) wqf on wqf.a_id = a.id and wqf.e_id = e.id

         left join (select e.id e_id, a.id a_id, count(*) fCount
                    from flips f
                             join transactions t on t.id = f.tx_id
                             join blocks b on b.id = t.block_id
                             join epochs e on e.id = b.epoch_id
                             join addresses a on a.id = t.from
                    where f.status is not NULL
                    group by a.id, e.id) allf on allf.a_id = a.id and allf.e_id = e.id
where e.epoch = $1